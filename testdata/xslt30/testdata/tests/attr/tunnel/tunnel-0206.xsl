<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:foo="http://www.foo.com"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             exclude-result-prefixes="foo"
             version="2.0">
<!-- Purpose: test empty base set within stylesheet function using xsl:apply-templates and tunnel="yes" -->

   <t:function name="foo:func">
		    <t:param name="thisNode"/>
		    <t:apply-templates select="$thisNode/data">
			      <t:with-param name="par" select="'xyz'" tunnel="yes"/>		
		    </t:apply-templates>	
	  </t:function>

   <t:template match="/">
		    <t:apply-templates select="doc">
			      <t:with-param name="par" select="'abc'" tunnel="yes"/>		
		    </t:apply-templates>
	  </t:template>

   <t:template match="doc">
		    <out>
         <t:value-of select="foo:func(self::doc)"/>
      </out>
	  </t:template>

   <t:template match="data">
		    <t:param name="par" select="'123'" tunnel="yes"/>		
		    <t:value-of select="$par"/>
	  </t:template>
</t:transform>
