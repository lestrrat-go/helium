<?xml version="1.0" encoding="iso-8859-1"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.3">

<?spec xslt#forwards?>
<?spec xslt#fallback?>
  <!-- PURPOSE: Test xsl:fallback for existing top-level 
       element used instruction. -->
  <xsl:template match="/">
    <out>
        <xsl:more-output href="xtemp.out">
             <xsl:fallback>Fallback processing</xsl:fallback>
        </xsl:more-output>
    </out>
  </xsl:template>
   
</xsl:stylesheet>
