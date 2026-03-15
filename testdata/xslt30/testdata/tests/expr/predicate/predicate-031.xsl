<?xml version="1.0"?>
<?spec xpath#id-predicates?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<xsl:output method="xml"/>

  <!-- FileName: predicate38 -->
  <!-- Document: http://www.w3.org/TR/xpath -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 3.4 -->
  <!-- Purpose: Stress test of nested and multiple predicates. The production rules
       allow such nesting. -->
  <!-- Author: Paul Dick -->
  
  <!-- priorities added by MHK to suppress warning messages -->

<xsl:template match="doc" priority="1">
<out>
  <predicate1>
   <xsl:apply-templates select="foo[(bar[2])='this']"/>
  </predicate1>
  <predicate2>
   <xsl:apply-templates select="foo[(bar[2][(baz[2])='goodbye'])]"/>
  </predicate2>
</out>
</xsl:template>

<xsl:template match="foo[(bar[2])='this']" priority="2">
 <xsl:text>1 is </xsl:text>	<xsl:for-each select="*">
    	<xsl:value-of select="@attr"/><xsl:text>,</xsl:text>
	</xsl:for-each>
</xsl:template>

<xsl:template match="foo[(bar[(baz[2])='goodbye'])]" priority="3">
 <xsl:text>3 is </xsl:text>	<xsl:for-each select="*">
    	<xsl:value-of select="@attr"/><xsl:text>,</xsl:text>
	</xsl:for-each>
</xsl:template>

<xsl:template match="foo[(bar[2][(baz[2])='goodbye'])]" priority="4">
 <xsl:text>2 is </xsl:text>	<xsl:for-each select="*">
    	<xsl:value-of select="@attr"/><xsl:text>,</xsl:text>
	</xsl:for-each>
</xsl:template>

</xsl:stylesheet>
