<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:one="http://mycharmap.example.org"
             xmlns:two="http://mycharmap.example.org"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             exclude-result-prefixes="one two"
             version="2.0">
<!-- Purpose: Test merging of two xsl:output declaration where the resulting 
				concatenated sequence of all use-character-maps is taken in declaration order.-->

   <t:output use-character-maps="two:format1"
             method="xml"
             indent="no"
             encoding="UTF-8"/>

   <t:output use-character-maps="one:format2"
             method="xml"
             indent="no"
             encoding="UTF-8"/>

   <t:character-map name="two:format2">
      <t:output-character character="#" string="&amp;"/>
      <t:output-character character="$" string="B"/>
      <t:output-character character="%" string="("/>
      <t:output-character character="%" string="*"/>
   </t:character-map>

   <t:character-map name="one:format1">
      <t:output-character character="$" string="A"/>
      <t:output-character character="%" string=")"/>
   </t:character-map>

   <t:template match="/doc">
  	   <out>
         <t:for-each select="*">
    		      <p>
               <t:value-of select="."/>
            </p>
            <t:text>
</t:text>
    	    </t:for-each>
      </out>
   </t:template>
</t:transform>
