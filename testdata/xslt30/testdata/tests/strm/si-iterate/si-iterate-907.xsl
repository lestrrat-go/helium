<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes=" xs"
    version="3.0">
   
  <!-- within xsl:stream, use xsl:iterate containing xsl:sequence -->
  <!-- Not streamable because the result of xsl:stream must not be a streamed node -->
   
  <xsl:template name="main">
    <out>
      <xsl:source-document streamable="yes" href="../docs/transactions.xml">
        <xsl:iterate select="account/transaction[@value &lt; 0]">
          <xsl:sequence select="."/>
        </xsl:iterate>
      </xsl:source-document>
    </out>
  </xsl:template> 

       
</xsl:transform>

