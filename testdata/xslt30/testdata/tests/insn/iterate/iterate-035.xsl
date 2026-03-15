<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  expand-text="yes"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:err="http://www.w3.org/2005/xqt-errors" 
  exclude-result-prefixes="#all">

  <!-- xsl:iterate with xsl:break within xsl:try -->
  
  <xsl:param name="z" select="1"/>

  <xsl:template match="/">
    <xsl:apply-templates select="//page">
      <xsl:with-param name="max" select="1000"/>
    </xsl:apply-templates>
  </xsl:template>

  <xsl:template match="page">
    <xsl:param name="max" as="xs:integer"/>
    <out>
      <xsl:iterate select="*">
        <xsl:choose>
          <xsl:when test="self::h3">
            <xsl:try>
              <pos>{position() idiv $z}</pos>
              <xsl:break/>
              <xsl:catch errors="err:FOAR0001">
                <continuing/>
              </xsl:catch>
            </xsl:try>           
          </xsl:when>
          <xsl:otherwise>
            <xsl:apply-templates select="."/>
          </xsl:otherwise>
        </xsl:choose>
      </xsl:iterate>
    </out>
  </xsl:template>

  <xsl:template match="p">
    <xsl:copy-of select="."/>
  </xsl:template>

</xsl:stylesheet>
