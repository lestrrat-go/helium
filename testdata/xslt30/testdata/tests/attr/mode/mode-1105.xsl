<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Set an initial context node, and have a template to match that node in mode X.
  				Where a named mode X is set at the time the transformation is launched. -->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:template match="/" mode="X">
	  <out>
	 	 <mode-X at="/"/>
	 	 <t:apply-templates select="doc" mode="a"/>
      </out>
   </t:template>

   <t:template match="doc" mode="X">
	  <out>
	     <mode-X at="doc"/>
	 	 <t:call-template name="my-temp"/>
      </out>
   </t:template>

   <t:template match="doc" mode="a">
	 	 <mode-a at="doc"/><t:call-template name="my-temp"/>
   </t:template>

   <t:template name="my-temp">
	  <inside><t:apply-templates select="str" mode="#current"/></inside>
   </t:template>

   <t:template match="str" mode="X">
	     <t:text>mode-X: </t:text>
	     <t:value-of select="."/>
   </t:template>
</t:transform>
