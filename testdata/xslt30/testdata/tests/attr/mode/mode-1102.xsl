<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Match a template whose mode list is X and other names, but not #default.
  				Where a named mode X is set at the time the transformation is launched. -->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:template match="doc" mode="b X  c ">
		    <out>
	 	 mode-bXc (doc): <t:call-template name="my-temp"/>
      </out>
	  </t:template>

   <t:template match="doc" mode=" a">
	  mode-a (doc): <t:call-template name="my-temp"/>
	  </t:template>

   <t:template name="my-temp">
	  inside named-templates: <t:apply-templates select="str" mode="#current"/>
	  </t:template>

   <t:template match="str" mode="a ">
	     <t:text>mode-a: </t:text>
	     <t:value-of select="."/>
	  </t:template>

   <t:template match="str" mode=" X ">
	     <t:text>mode-X: </t:text>
	     <t:value-of select="."/>
	  </t:template>
</t:transform>
