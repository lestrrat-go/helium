<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:foo="http://www.foo.com"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             version="2.0">
<!-- Purpose: Test case to show a pseudo-clash between the name of a param passed explicitly 
                and a tunnel param from higher up that is not mentioned at either level of 
                explicit pass but is used lower down.  -->

   <t:function name="foo:func">
		    <t:call-template name="temp1">
			      <t:with-param name="par1" select="'xyz'"/>		
		    </t:call-template>	
	  </t:function>

   <t:template name="temp1">
		    <t:param name="par1" select="'123'"/>
		    <t:value-of select="$par1"/>
	  </t:template>

   <t:template match="/">
  	   <t:text>
</t:text>
		    <out>
         <t:apply-templates select="doc">
			         <t:with-param name="par1" select="'foo'" tunnel="yes"/>
		       </t:apply-templates>
         <t:text>
</t:text>
      </out>
	  </t:template>

   <t:template match="doc">
		    <t:apply-templates select="data"/>
	  </t:template>

   <t:template match="data">
		    <t:apply-templates select="inner"/>
	  </t:template>

   <t:template match="inner">
		    <t:param name="par1" select="'xyz'" tunnel="yes"/>
		par1:<t:value-of select="$par1"/>  
	  </t:template>
</t:transform>
