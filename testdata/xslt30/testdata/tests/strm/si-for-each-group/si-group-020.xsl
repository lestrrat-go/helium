<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes=" xs"
    version="3.0">
   
  <!-- within xsl:stream, use xsl:for-each-group group-adjacent, position reference to group only-->

  <xsl:template name="main">
    <xsl:source-document streamable="yes" href="../docs/transactions.xml">
     <out>
      <xsl:for-each-group select="/account/transaction"
         group-adjacent="@date">
         <g pos="{position()}"/>
      </xsl:for-each-group> 
     </out>
    </xsl:source-document>
  </xsl:template> 

       
</xsl:transform>

