<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:foo="http://www.foo.com"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             exclude-result-prefixes="foo"
             version="2.0">
<!-- Purpose: test empty base set within stylesheet function using call-template -->

   <t:function name="foo:func">
		    <t:call-template name="temp1">
			      <t:with-param name="par" select="'xyz'" tunnel="yes"/>		
		    </t:call-template>	
	  </t:function>

   <t:template match="/">
		    <t:apply-templates select="doc">
			      <t:with-param name="par" select="'abc'" tunnel="yes"/>		
		    </t:apply-templates>
	  </t:template>

   <t:template match="doc">
		    <out>
         <t:value-of select="foo:func()"/>
      </out>
	  </t:template>

   <t:template name="temp1">
		    <t:param name="par" select="'123'" tunnel="yes"/>		
		    <t:value-of select="$par"/>
	  </t:template>
</t:transform>
