<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:mode streamable="yes"/>
  <xsl:output method="xml"/>
  
  <!-- This test is designed to exercise implementations that attempt to be smarter in their
    strategy for rule evaluation than simply testing each rule in turn. -->
  
  <xsl:template match="email/text()">
    <xsl:value-of select="current()"/>
  </xsl:template>
  <xsl:template match="email/text()[. = 'abcde']"/>
  <xsl:template match="email/text()[. = 'abcde2']"/>
  <xsl:template match="email/text()[. = 'abcde3']"/>
  <xsl:template match="email/text()[. = 'abcde4']"/>
  <xsl:template match="email/text()[. = 'abcde5']"/>
  <xsl:template match="email/text()[. = 'abcde6']"/>
  
  <xsl:template match="address/text()">
    <xsl:value-of select="current()"/>
  </xsl:template>
  <xsl:template match="address/text()[. = 'abcde']"/>
  <xsl:template match="address/text()[. = 'abcde2']"/>
  <xsl:template match="address/text()[. = 'abcde3']"/>
  <xsl:template match="address/text()[. = 'abcde4']"/>
  <xsl:template match="address/text()[. = 'abcde5']"/>
  <xsl:template match="address/text()[. = 'abcde6']"/>
  <xsl:template match="address/text()[. = 'abcde7']"/>
  
  <xsl:template match="birthday/text()">
    <xsl:value-of select="current()"/>
  </xsl:template>
  <xsl:template match="birthday/text()[. = 'abcde']"/>
  <xsl:template match="birthday/text()[. = 'abcde2']"/>
  <xsl:template match="birthday/text()[. = 'abcde3']"/>
  <xsl:template match="birthday/text()[. = 'abcde4']"/>
  <xsl:template match="birthday/text()[. = 'abcde5']"/>
  <xsl:template match="birthday/text()[. = 'abcde6']"/>
  <xsl:template match="birthday/text()[. = 'abcde7']"/>
  
  <xsl:template match="country/text()">
    <xsl:value-of select="current()"/>
  </xsl:template>
  <xsl:template match="country/text()[. = 'abcde']"/>
  <xsl:template match="country/text()[. = 'abcde2']"/>
  <xsl:template match="country/text()[. = 'abcde3']"/>
  <xsl:template match="country/text()[. = 'abcde4']"/>
  <xsl:template match="country/text()[. = 'abcde5']"/>
  <xsl:template match="country/text()[. = 'abcde6']"/>
  <xsl:template match="country/text()[. = 'abcde7']"/>
  <xsl:template match="country/text()[. = 'abcde8']"/>
  
  <xsl:template match="external/text()">
    <xsl:value-of select="current()"/>
  </xsl:template>
  <xsl:template match="external/text()[. = 'abcde']"/>
  <xsl:template match="external/text()[. = 'abcde2']"/>
  <xsl:template match="external/text()[. = 'abcde3']"/>
  <xsl:template match="external/text()[. = 'abcde4']"/>
  <xsl:template match="external/text()[. = 'abcde5']"/>
  <xsl:template match="external/text()[. = 'abcde6']"/>
  
  <xsl:template match="internal/text()">
    <xsl:value-of select="current()"/>
  </xsl:template>
  <xsl:template match="internal/text()[. = 'abcde']"/>
  <xsl:template match="internal/text()[. = 'abcde2']"/>
  <xsl:template match="internal/text()[. = 'abcde3']"/>
  <xsl:template match="internal/text()[. = 'abcde4']"/>
  <xsl:template match="internal/text()[. = 'abcde5']"/>
  <xsl:template match="internal/text()[. = 'abcde6']"/>
  
  <xsl:template match="name/text()">
    <xsl:value-of select="current()"/>
  </xsl:template>
  <xsl:template match="name/text()[. = 'abcde']"/>
  <xsl:template match="name/text()[. = 'abcde2']"/>
  <xsl:template match="name/text()[. = 'abcde3']"/>
  <xsl:template match="name/text()[. = 'abcde4']"/>
  <xsl:template match="name/text()[. = 'abcde5']"/>
  <xsl:template match="name/text()[. = 'abcde6']"/>
  
  <xsl:template match="phone/text()">
    <xsl:value-of select="current()"/>
  </xsl:template>
  <xsl:template match="phone/text()[. = 'abcde']"/>
  <xsl:template match="phone/text()[. = 'abcde2']"/>
  <xsl:template match="phone/text()[. = 'abcde3']"/>
  <xsl:template match="phone/text()[. = 'abcde4']"/>
  <xsl:template match="phone/text()[. = 'abcde5']"/>
  <xsl:template match="phone/text()[. = 'abcde6']"/>
  
  <xsl:template match="city/text()">
    <xsl:value-of select="current()"/>
  </xsl:template>
  <xsl:template match="city/text()[. = 'abcde']"/>
  <xsl:template match="city/text()[. = 'abcde2']"/>
  <xsl:template match="city/text()[. = 'abcde3']"/>
  <xsl:template match="city/text()[. = 'abcde4']"/>
  <xsl:template match="city/text()[. = 'abcde5']"/>
  <xsl:template match="city/text()[. = 'abcde6']"/>
  
  
  <xsl:template match="id/text()">
    <xsl:value-of select="concat(current(), '_ID')"/>
  </xsl:template>
  <xsl:template match="id/text()[. = 'abcde']"/>
  <xsl:template match="id/text()[. = 'abcde2']"/>
  <xsl:template match="id/text()[. = 'abcde3']"/>
  <xsl:template match="id/text()[. = 'abcde4']"/>
  <xsl:template match="id/text()[. = 'abcde5']"/>
  <xsl:template match="id/text()[. = 'abcde6']"/>
  
  
  <xsl:template match=".">
    <xsl:copy validation="preserve">
      <xsl:apply-templates select="@*"/>
      <xsl:apply-templates select="node()"/>
    </xsl:copy>
  </xsl:template>
</xsl:stylesheet>
