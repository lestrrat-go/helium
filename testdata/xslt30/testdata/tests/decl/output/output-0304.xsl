<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:one="http://mycharmap.example.org"
             xmlns:two="http://mycharmap.example.org"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             exclude-result-prefixes="one two"
             version="2.0">
<!-- Purpose: Test  @use-character-maps on xsl:result-document, 
				referencing a character map with prefixed name  by using 
				a different prefix bound to the same namespace.-->

   <t:output name="outputDoc" method="xml" indent="no" encoding="UTF-8"/>

   <t:character-map name="one:format1">
      <t:output-character character="$" string="A"/>
   </t:character-map>

   <t:character-map name="two:format2">
      <t:output-character character="#" string="&amp;"/>
   </t:character-map>

   <t:template match="/doc">
      <t:result-document format="outputDoc" use-character-maps="two:format1 one:format2">
  	      <out>
            <t:for-each select="*">
    		         <p>
                  <t:value-of select="."/>
               </p>
               <t:text>
</t:text>
    	       </t:for-each>
         </out>
      </t:result-document>
   </t:template>
</t:transform>
