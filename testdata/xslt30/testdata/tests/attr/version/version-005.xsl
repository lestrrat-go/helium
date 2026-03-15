<?xml version="1.0" encoding="iso-8859-1"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="8.5">

  <!-- PURPOSE: Test xsl:fallback for an extension element. -->
  <?spec xslt#fallback?>
  <xsl:template match="/">
    <out xsl:extension-element-prefixes="burroughs honeywell cdc"
        xmlns:burroughs="http://www.burroughs.com/"
        xmlns:honeywell="http://www.honeywell.com/"
        xmlns:cdc="http://www.cdc.com/">
        <cdc:non-existent-element processing="magic">
             <bananas/>
             <custard/>
             <pie>PIE!</pie>
             <xsl:fallback>Fallback processing </xsl:fallback>
             <xsl:fallback>More fallback processing</xsl:fallback>
        </cdc:non-existent-element>
    </out>
  </xsl:template>
   
</xsl:stylesheet>
