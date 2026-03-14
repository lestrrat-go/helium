<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

  <!-- xsl:iterate; passing two parameters where the new value of the second param depends on the old value of the first;
       and the first is consuming-->

  <xsl:template name="main">
    <xsl:source-document streamable="yes" href="../docs/transactions.xml">
      <out>
        <xsl:iterate select="/*/transaction">
          <xsl:param name="first" select="1" as="xs:integer"/>
          <xsl:param name="second" select="0" as="xs:integer"/>
          <xsl:param name="prev" select="()"/>
          <x one="{$first}" two="{$second}" value="{$prev/@value}"/>
          <xsl:next-iteration>
            <xsl:with-param name="first" select="$first + 1"/>
            <xsl:with-param name="prev" select="copy-of(.)[$first gt 3]"/>
            <xsl:with-param name="second" select="$first"/>             
          </xsl:next-iteration>
        </xsl:iterate>
      </out>
    </xsl:source-document>    
  </xsl:template>
  
</xsl:stylesheet>
