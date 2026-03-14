<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

  <!-- xsl:iterate; example from spec (adapted) to show special handling of the last item in a sequence,
       in a way that works with streaming -->

  <xsl:template name="main">
    <xsl:source-document streamable="yes" href="../docs/transactions.xml">
      <out>
        <xsl:iterate select="/*/transaction">
          <xsl:param name="prev" select="()" as="element(transaction)?"/>
          <xsl:on-completion>
            <y><xsl:copy-of select="$prev"/></y>      
          </xsl:on-completion>
          <xsl:if test="$prev">
            <x><xsl:copy-of select="$prev"/></x>
          </xsl:if>
          <xsl:next-iteration>
            <xsl:with-param name="prev" select="copy-of(.)"/>
          </xsl:next-iteration>
        </xsl:iterate>
      </out>
    </xsl:source-document>    
  </xsl:template>
  
</xsl:stylesheet>
