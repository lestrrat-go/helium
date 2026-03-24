<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:mode streamable="yes"/>
  <xsl:output method="xml"/>
  <xsl:template match="email/text()">
    <xsl:value-of select="current()"/>
  </xsl:template>
  <xsl:template match="address/text()">
    <xsl:value-of select="current()"/>
  </xsl:template>
  <xsl:template match="birthday/text()">
    <xsl:value-of select="current()"/>
  </xsl:template>
  <xsl:template match="country/text()">
    <xsl:value-of select="current()"/>
  </xsl:template>
  <xsl:template match="external/text()">
    <xsl:value-of select="current()"/>
  </xsl:template>
  <xsl:template match="internal/text()">
    <xsl:value-of select="current()"/>
  </xsl:template>
  <xsl:template match="name/text()">
    <xsl:value-of select="current()"/>
  </xsl:template>
  <xsl:template match="phone/text()">
    <xsl:value-of select="current()"/>
  </xsl:template>
  <xsl:template match="city/text()">
    <xsl:value-of select="current()"/>
  </xsl:template>
  <xsl:template match="id/text()">
    <xsl:value-of select="concat(current(), '_ID')"/>
  </xsl:template>
  <xsl:template match=".">
    <xsl:copy validation="preserve">
      <xsl:apply-templates select="@*"/>
      <xsl:apply-templates select="node()"/>
    </xsl:copy>
  </xsl:template>
</xsl:stylesheet>
