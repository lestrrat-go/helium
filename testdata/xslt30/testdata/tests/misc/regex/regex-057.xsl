<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Testcase with replace($input?,$pattern,$replacement) on a literal string or empty 
  				sequence where pattern uses positional metacharacters ^ and $. 
				Included border cases:  - $input is an empty string
										- $input is an empty sequence
										- $pattern does not find a match -->

   <t:output method="xml" encoding="UTF-8"/>

   <t:template match="/">
      <out>
         
         <E1>
            <t:value-of select="replace('abracadabra','bra', '*')"/>
         </E1>
         
         <E2>
            <t:value-of select="replace('abracadabra','^a.*a$', '*')"/>
         </E2>
         
         <E3>
            <t:value-of select="replace('abracadabra','^.+$', '*')"/>
         </E3>
         
         <E4>
            <t:value-of select="replace('abracadabra','^.+.*.*$', '*')"/>
         </E4>
         
         <E5>
            <t:value-of select="replace('','^ $', '*')"/>
         </E5>
         
         <E6>
            <t:value-of select="replace((),'^ $', '*')"/>
         </E6>
         
         <E7>
            <t:value-of select="replace('abracadabra','^a(.)','*$1$1')"/>
         </E7>
         
         <E8>
            <t:value-of select="replace('abracadabra','^A.*A$', '*')"/>
         </E8>
         
         <E9>
            <t:value-of select="replace('abracadabra','^bra', '*')"/>
         </E9>
         
         <E10>
            <t:value-of select="replace('abracadabra','abr$', '*')"/>
         </E10>
         
         <E11>
            <t:value-of select="replace('abracadabra','^.$', '*')"/>
         </E11>
         
      </out>
   </t:template>
</t:transform>
