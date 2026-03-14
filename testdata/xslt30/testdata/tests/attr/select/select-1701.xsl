<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:xslt="http://www.w3.org/1999/XSL/Transform" version="2.0" exclude-result-prefixes="xs">
  <!-- Purpose: Test of select in xsl:apply-templates with complex pathExpr involving various node and name tests -->

  <xslt:template match="doc">
    <out>
      <xslt:apply-templates select="(../element()/attribute())[string(.) ne 'at2']"/>
    </out>
  </xslt:template>

  <xslt:template match="@*">
    <xslt:choose>
      <xslt:when test="name(../child::element()/attribute()[string(.) ne 'at2'][1]) = 'att1'">
        <xslt:value-of select="name(.)"/>
        <xslt:text>|</xslt:text>
        <xslt:apply-templates select="../*:num1/following::element()"/>
      </xslt:when>
      <xslt:otherwise>
        <xslt:value-of select="name(reverse(.))"/>
        <xslt:text>|</xslt:text>
        <xslt:apply-templates select="../*:num1/following::element()"/>
      </xslt:otherwise>
    </xslt:choose>
  </xslt:template>

  <xslt:template match="*">
    <xslt:value-of select="name(.)"/>
    <xslt:text>|</xslt:text>
  </xslt:template>

  <xslt:template match="text()"/>
</xslt:transform>
