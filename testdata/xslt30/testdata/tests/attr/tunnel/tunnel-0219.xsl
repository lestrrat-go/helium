<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:foo="http://www.foo.com"
             xmlns:bar="http://www.bar.com"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             exclude-result-prefixes="foo bar"
             version="2.0">
<!-- Purpose: Tests tunnel parameters with the same local name 
       but different namespace specified within xsl:apply-imports. -->

   <t:import href="import1.xsl"/>

   <t:template match="doc">
		    <out>
         <t:apply-imports>
			         <t:with-param name="foo:par1" select="'abc'" tunnel="yes"/>	
			         <t:with-param name="bar:par2" select="'gta'" tunnel="yes"/>	
		       </t:apply-imports>
      </out>
	  </t:template>

   <t:template match="data">
		    <t:param name="foo:par1" select="'123'" tunnel="yes"/>
		    <t:param name="bar:par2" select="'346'" tunnel="yes"/>
		    <t:param name="bar:par1" select="'678'" tunnel="yes"/>
		    <t:param name="foo:par2" select="'555'" tunnel="yes"/>		
		    <t:value-of select="$foo:par1"/> and <t:value-of select="$bar:par1"/> and <t:value-of select="$foo:par2"/> and <t:value-of select="$bar:par2"/>
	  </t:template>

   <t:template match="text()"/>
</t:transform>
