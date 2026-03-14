<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:cat="http://www.w3.org/2012/10/xslt-test-catalog"
  exclude-result-prefixes="cat">
  
  <!-- assert-xml must not be used where indent=yes is specified -->

<xsl:template match="/">
      <out>
        <xsl:for-each select="/*/cat:test-set/document(@file)/*/cat:test-case">
          <xsl:variable name="tc" select="."/>
          <xsl:if test="cat:test/cat:stylesheet/document(@file)/*/xsl:output[@indent='yes']
                          and $tc/cat:result//cat:assert-xml">
            <indenting><xsl:value-of select="@name"/></indenting>
          </xsl:if>
        </xsl:for-each>  
      </out>
</xsl:template>

</xsl:stylesheet>