<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

  <!-- Ensure param of xsl:iterate evaluated in correct context -->


  <xsl:template match="/">
    <xsl:apply-templates select="//page">
      <xsl:with-param name="max" select="1000"/>
    </xsl:apply-templates>
  </xsl:template>

  <xsl:template match="page">
    <xsl:param name="max" as="xs:integer"/>
    <xsl:param name="page" select="."/>
    <out>
      <xsl:for-each select="5 to 7">
        <loop n="{.}">
          <xsl:iterate select="$page/*">
            <xsl:param name="c" select="."/>
            <xsl:param name="p" select="position()"/>
            <z c="{$c}" p="{$p}"/>
            <xsl:choose>
              <xsl:when test="self::h3">
                <xsl:break>
                  <exit at="{position()}" of="{last()}"/>
                </xsl:break>
              </xsl:when>
              <xsl:otherwise>
                <xsl:apply-templates select="."/>
                <xsl:next-iteration>
                  <xsl:with-param name="p" select="$p + 1"/>
                  <xsl:with-param name="c" select="$c + 1"/>
                </xsl:next-iteration>
              </xsl:otherwise>
            </xsl:choose>
          </xsl:iterate>
        </loop>
      </xsl:for-each>
    </out>
  </xsl:template>

  <xsl:template match="p">
    <xsl:copy-of select="."/>
  </xsl:template>

</xsl:stylesheet>
