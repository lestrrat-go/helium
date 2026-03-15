<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Testcase with replace($input?,$pattern,$replacement) on a literal string or empty sequence 
  				where pattern uses reluctant quantifiers. Included border cases:
				- $input is an empty string
				- $input is an empty sequence
				- $pattern does not find a match-->

   
   <t:template match="/">
      
      <out>
         
         <e1>
            <t:value-of select="replace('aaaa','a+', 'b')"/>
         </e1>
         
         <E1>
            <t:value-of select="replace('aaaa','a+?', 'b')"/>
         </E1>
         
         <E2>
            <t:value-of select="replace('aaaaaa','a{2}?', 'b')"/>
         </E2>
         
         <e3>
            <t:value-of select="replace('aaaaaa','a{2,}', 'b')"/>
         </e3>
         
         <E3>
            <t:value-of select="replace('aaaaaa','a{2,}?', 'b')"/>
         </E3>
         
         <e4>
            <t:value-of select="replace('aaaaaa','a{2,3}', 'b')"/>
         </e4>
         
         <E4>
            <t:value-of select="replace('aaaaaa','a{2,3}?', 'b')"/>
         </E4>
         
         <E5>
            <t:value-of select="replace('a','a{2,3}?', 'b')"/>
         </E5>
         
         <E6>
            <t:value-of select="replace('','.+?', 'b')"/>
         </E6>
         
         <E7>
            <t:value-of select="replace((),'.+?', 'b')"/>
         </E7>
         
      </out>
   </t:template>
</t:transform>
