<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
version="2.0">

<?spec fo#func-id?>
<!-- Throws NullPointerException at 6.4.4 -->

  <xsl:template match="/">
    <result>
      <xsl:for-each select="//person[@job = (../@job |
id(../@spouse)/@job)]">
        <xsl:copy>
          <xsl:copy-of select="@*"/>
        </xsl:copy>
      </xsl:for-each>
    </result>
  </xsl:template>

</xsl:transform>
