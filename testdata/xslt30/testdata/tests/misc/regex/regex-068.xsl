<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
   <!-- Purpose: Testcase with replace($input?,$pattern,$replacement,$flags) using multiline mode ($flags is set to 'm'). -->

   <t:output method="xml" encoding="UTF-8"/>

   <t:template match="/doc">
      <output>
         
         <E1>
            <t:value-of select="replace(poem,'Kaum.*krahen', '*')"/>
         </E1>

         <E2>
            <t:value-of select="replace(poem_multiline,'Kaum.*krahen', '*')"/>
         </E2>

         <E3>
            <t:value-of select="replace(poem_multiline,'Kaum.*krahen', '*', 'm')"/>
         </E3>

         <E4>
            <t:value-of select="replace(poem,'^Kaum.*gesehen,$', '*')"/>
         </E4>

         <E5>
            <t:value-of select="replace(poem_multiline,'^Kaum.*gesehen,$', '*')"/>
         </E5>

         <E6>
            <t:value-of select="replace(poem_multiline,'^Kaum.*gesehen,$', '*','m')"/>
         </E6>

      </output>
   </t:template>
</t:transform>
