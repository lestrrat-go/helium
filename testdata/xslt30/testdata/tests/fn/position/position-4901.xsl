<?xml version="1.0" encoding="UTF-8"?>
<?spec fo#func-position?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Purpose: Data Model requires that namespace nodes precede attribute nodes. -->

<xsl:output encoding="UTF-8" method="xml" indent="no"/>

<xsl:template match="Doc">
  <out>
    <xsl:for-each select="namespace::* | attribute::*" >
      <xsl:choose>
        <xsl:when test="contains(.,'http')">
          <!-- it's a namespace node -->
          <xsl:if test="position() &lt;= 3">
            <xsl:text> OK</xsl:text>
          </xsl:if>
        </xsl:when>
        <xsl:when test="contains(.,'attr')">
          <!-- it's an attribute node -->
          <xsl:if test="position() &gt; 3">
            <xsl:text> OK</xsl:text>
          </xsl:if>
        </xsl:when>
        <xsl:otherwise>
          <xsl:text>&#10;BAD VALUE: </xsl:text><xsl:value-of select="."/>
        </xsl:otherwise>
      </xsl:choose>
    </xsl:for-each>
  </out>
</xsl:template>

</xsl:stylesheet>