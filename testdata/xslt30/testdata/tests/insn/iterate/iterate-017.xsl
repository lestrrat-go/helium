<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

  <!-- Test of xsl:iterate; params not given a value retain value from previous iteration -->

  <xsl:template match="/">
    <xsl:apply-templates select="//page">
      <xsl:with-param name="max" select="1000"/>
    </xsl:apply-templates>
  </xsl:template>

  <xsl:template match="page">
    <xsl:param name="max" as="xs:integer"/>
    <out>
      <xsl:iterate select="*">
        <xsl:param name="count-h3" select="0"/>
        <xsl:param name="count-other" select="0"/>
        <xsl:on-completion>
          <count h3="{$count-h3}" other="{$count-other}"/>
        </xsl:on-completion>
        <xsl:choose>
          <xsl:when test="self::h3">
            <heading>
              <xsl:copy-of select="child::node()"/>
            </heading>
            <xsl:next-iteration>
              <xsl:with-param name="count-h3" select="$count-h3 + 1"/>
            </xsl:next-iteration>
          </xsl:when>
          <xsl:otherwise>
            <xsl:element name="{name()}"/>
            <xsl:next-iteration>
              <xsl:with-param name="count-other" select="$count-other + 1"/>
            </xsl:next-iteration>
          </xsl:otherwise>
        </xsl:choose>
        <xsl:fallback/>
      </xsl:iterate>
    </out>
  </xsl:template>

  <xsl:template match="p">
    <xsl:copy-of select="."/>
  </xsl:template>

</xsl:stylesheet>
