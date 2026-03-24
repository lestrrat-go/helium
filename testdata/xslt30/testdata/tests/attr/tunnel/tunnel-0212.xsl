<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Tests the name clash between tunnel parameter and explicit parameter defined in templates.
  The tunnel parameter is silently passed through intermediate templates although its name is similar to the
  parameter defined in the intermediate template. -->

   <t:template name="A" match="/">
		    <out>
         <t:call-template name="B">
			         <t:with-param name="par1" select="'foo'" tunnel="yes"/>
			         <t:with-param name="par2" select="'bar'" tunnel="yes"/>
		       </t:call-template>
      </out>
	  </t:template>

   <t:template name="B">
		    <t:param name="par1" select="'nametest1'"/>
		    <t:call-template name="C">
		</t:call-template>
	  </t:template>

   <t:template name="C">
		    <t:param name="par2" select="'nametest2'"/>
		    <t:call-template name="D"/>
	  </t:template>

   <t:template name="D">
		    <t:param name="par1" select="'xyz'" tunnel="yes"/>
		    <t:param name="par2" select="'zyx'" tunnel="no"/>
		    <t:value-of select="$par1"/> and <t:value-of select="$par2"/>  	
	  </t:template>
</t:transform>
