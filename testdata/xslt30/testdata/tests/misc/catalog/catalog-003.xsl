<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:cat="http://www.w3.org/2012/10/xslt-test-catalog"
  exclude-result-prefixes="cat">
  
  <!-- Test case names must be unique -->

<xsl:template match="/">
      <out>
        <xsl:for-each-group select="/*/cat:test-set/document(@file)/*/cat:test-case" group-by="@name">
          <xsl:if test="count(current-group()) ne 1">
            <duplicate><xsl:value-of select="current-grouping-key()"/></duplicate>
          </xsl:if>
        </xsl:for-each-group>  
      </out>
</xsl:template>

</xsl:stylesheet>