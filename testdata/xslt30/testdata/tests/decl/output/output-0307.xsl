<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test with xsl:output that has @use-character-maps and 
				an xsl:result-document that uses this output declaration 
				but includes @use-character-maps on its own. Show that the 
				charactes-maps references in xsl:result-document supplement 
				and take precedence over the ones from the xsl:output declaration.-->

   <t:output name="temp"
             use-character-maps="format1"
             method="xml"
             indent="no"
             encoding="UTF-8"/>

   <t:character-map name="format1">
      <t:output-character character="$" string="B"/>
      <t:output-character character="%" string="("/>
      <t:output-character character="%" string="*"/>
   </t:character-map>

   <t:character-map name="format2">
      <t:output-character character="#" string="-"/>
      <t:output-character character="$" string="A"/>
      <t:output-character character="%" string=")"/>
   </t:character-map>

   <t:template match="/doc">
      <t:result-document format="temp" use-character-maps="format2">
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
