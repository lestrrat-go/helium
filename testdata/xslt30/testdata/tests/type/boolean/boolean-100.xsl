<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#id-general-comparisons?>
    <!-- Purpose: If $x is bound to a node-set, then $x="foo" does not 
       mean the same as not($x!="foo"): the former is true if and only 
       if some node in $x has the string-value foo; the latter is true if 
       and only if all nodes in $x have the string-value foo. -->

<xsl:template match="doc">
	<out>

	  <xsl:variable name="x" select="av//*"/>
	  <xsl:value-of select="$x='foo'"/>
	  <xsl:value-of select="not($x!='foo')"/><xsl:text> </xsl:text>

	  <xsl:variable name="y" select="av//j"/>
	  <xsl:value-of select="$y='foo'"/>
	  <xsl:value-of select="not($y!='foo')"/>

	</out>
</xsl:template>

</xsl:stylesheet>