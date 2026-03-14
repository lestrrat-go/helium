<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:err="http://www.w3.org/2005/xqt-errors" exclude-result-prefixes="err">

  <!-- xsl:catch with variables to receive error information -->

  <xsl:template name="main">
    <out>
      <xsl:try>
        <xsl:sequence select="doc('try-018-rubbish.xml')"/>
        <xsl:catch errors="*">
          <code ns="{namespace-uri-from-QName($err:code)}" local="{local-name-from-QName($err:code)}"/>
          <description><xsl:value-of select="$err:description"/></description>
          <value><xsl:value-of select="$err:value"/></value>
          <module><xsl:value-of select="$err:module"/></module>
          <line><xsl:value-of select="$err:line-number"/></line>
          <column><xsl:value-of select="$err:column-number"/></column>
        </xsl:catch>
      </xsl:try>
    </out>
  </xsl:template>

</xsl:stylesheet>
