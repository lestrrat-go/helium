<?xml version="1.0"?>
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:t="http://ns.example.com/val12/">

  <!-- show templates matching against validated elements from the source document -->


  <xsl:import-schema namespace="http://ns.example.com/val12/" schema-location="validation-12.xsd"/>

  <xsl:template name="main">
    <z>
      <xsl:apply-templates select="doc('validation-20.xml')/*"/>
      <xsl:apply-templates select="doc('validation-20a.xml')/*"/>
    </z>
  </xsl:template>

  <xsl:template match="schema-element(t:test)" priority="3">
    <a>schema-element() matches when nilled=<xsl:value-of select="nilled(.)"/></a>
  </xsl:template>

  <xsl:template match="element(t:test)" priority="2">
    <a>schema-element doesn't match when nilled=<xsl:value-of select="nilled(.)"/></a>
  </xsl:template>

</xsl:stylesheet>
