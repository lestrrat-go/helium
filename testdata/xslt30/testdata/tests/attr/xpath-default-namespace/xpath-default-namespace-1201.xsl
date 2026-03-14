  <xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                  version="2.0">

<?spec xslt#built-in-rule?>                  
    <!-- PURPOSE: test whether parameter values are passed unchanged through a built-in template -->
    <?same-as-1.0 no?>              

    <xsl:template match="a">
      <xsl:apply-templates>
        <xsl:with-param name="flavor">mint</xsl:with-param>
      </xsl:apply-templates>
    </xsl:template>

    <xsl:template match="c">
      <xsl:param name="flavor">vanilla</xsl:param>
      <z>Today's flavor: <xsl:value-of select="$flavor"/></z>
    </xsl:template>

  </xsl:stylesheet>
