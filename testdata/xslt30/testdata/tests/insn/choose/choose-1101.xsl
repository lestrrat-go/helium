<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#xsl-if?>
    <!-- Purpose: Test that xsl:if can be nested. -->

<xsl:template match="letters">
  <out>
    <xsl:apply-templates select="letter"/>
  </out>
</xsl:template>

<xsl:template match="letter">
  <xsl:if test="2+2 = 4">
    <xsl:if test="'d'='d'">
      <xsl:if test=".='b'">
        <xsl:if test="0 = 0">
          <xsl:if test=".='b'">
            <xsl:if test="name(..)='letters'">
              <xsl:if test="1+1 = 2">
                <xsl:text>Found b!</xsl:text>
              </xsl:if>
            </xsl:if>
          </xsl:if>
        </xsl:if>
      </xsl:if>
    </xsl:if>
  </xsl:if>
</xsl:template>

</xsl:stylesheet>
