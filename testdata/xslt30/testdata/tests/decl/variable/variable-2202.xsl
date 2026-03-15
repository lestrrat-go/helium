<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

 <?spec xslt#with-param?>
  <!-- Purpose: It is not an error to pass a parameter to a template that does
       not have an element for it, the parameter is simpily ignored. -->
 
  
  <!-- This is an error in XSLT 2.0 if the stylesheet specifies version=2.0, but not with version="1.0" -->
  <?same-as-1.0 no?>

<xsl:output method="xml"/>

<xsl:param name="test" select="'global'"/>

<xsl:template match="/">
  <out>
    <xsl:call-template name="temtest">
      <xsl:with-param name="test" select="'local'"/>
    </xsl:call-template>
  </out>
</xsl:template>

<xsl:template name="temtest">
  <xsl:choose>
    <xsl:when test="$test = 'global'">It is global!</xsl:when>
    <xsl:otherwise>Not global!!!</xsl:otherwise>
  </xsl:choose>
</xsl:template>

</xsl:stylesheet>