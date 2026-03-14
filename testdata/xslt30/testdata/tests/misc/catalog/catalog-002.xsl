<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:cat="http://www.w3.org/2012/10/xslt-test-catalog"
  exclude-result-prefixes="cat">

<xsl:template match="/">
      <ok>
        <xsl:for-each select="/*/cat:test-set">
          <xsl:variable name="name" select="@name"/>
          <xsl:if test="not(document(@file)/*/@name = $name)">
            <mismatch master="{$name}" slave="{document(@file)/*/@name}"/>
          </xsl:if>
        </xsl:for-each>  
      </ok>
</xsl:template>

</xsl:stylesheet>