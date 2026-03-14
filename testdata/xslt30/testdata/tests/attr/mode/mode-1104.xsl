<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Set stylesheet params, at least one of which will be used in the match="/" mode="X" template.
  				Where a named mode X is set at the time the transformation is launched. -->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:param name="Sparam" select="'defaultValue'"/>

   <t:template match="/" mode="X">
		    <out>
	 	 mode-X (/): <t:value-of select="$Sparam"/>
         <t:apply-templates select="doc" mode="#current"/>
      </out>
	  </t:template>

   <t:template match="doc" mode="#all">
	 	 mode-#all (doc): <t:call-template name="my-temp"/>
	  </t:template>

   <t:template name="my-temp">
	  inside named-templates: <t:apply-templates select="str" mode="X"/>
	  </t:template>

   <t:template match="str" mode="X">
	     <t:text>mode-X: </t:text>
	     <t:value-of select="."/>
	  </t:template>
</t:transform>
