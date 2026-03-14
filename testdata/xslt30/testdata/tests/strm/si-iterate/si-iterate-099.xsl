<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
  xmlns:xs="http://www.w3.org/2001/XMLSchema" 
  xmlns:math="http://www.w3.org/2005/xpath-functions/math" exclude-result-prefixes="xs math" 
  version="3.0">
  
  <xsl:output method="xml"/>
  
  <xsl:param name="input-uri" as="xs:string" select="'test2016091202.xml'"/>
  
  <xsl:variable name="contains-debit" as="xs:boolean">
    <xsl:source-document href="../docs/transactions.xml" streamable="yes">
      <xsl:iterate select=".//transaction">
        <xsl:on-completion select="false()"/>
        <xsl:if test="@value &lt; 0">
          <xsl:break select="true()"/>
        </xsl:if>
      </xsl:iterate>
    </xsl:source-document>
  </xsl:variable>
  
  <xsl:template name="xsl:initial-template">
    <out><xsl:value-of select="$contains-debit"/></out>
  </xsl:template>
  
</xsl:stylesheet>

