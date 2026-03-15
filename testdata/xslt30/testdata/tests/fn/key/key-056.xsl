<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">


  <xsl:key name="primary" match="indexterm" use="primary"/>
  <xsl:key name="primary" match="indexterm" use="primary"/>

  <xsl:template match="/">
    <doc>
      <!-- select every primary keyword only once -->
      <xsl:apply-templates select="//indexterm[count(.|key('primary',primary)[1])= 1]"/>
    </doc>
  </xsl:template>

  <xsl:template match="indexterm">
    <!-- this template should be invoked once for every keyword -->
    <val for="{primary}"><xsl:value-of select="count(key('primary',primary))"/></val>
  </xsl:template>
</xsl:stylesheet>
