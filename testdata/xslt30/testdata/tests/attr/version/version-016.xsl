<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

<!-- PURPOSE:  Test trace() function with a sequence of nodes -->
<?spec fo#func-trace?>

<xsl:template match="doc">
<out>
  <xsl:for-each select="trace(//*, 'desc-element')">
     <e><xsl:value-of select="name(.)"/></e>
       <xsl:for-each select="trace(@*, 'att')">
          <a><xsl:value-of select="name(.)"/></a>
       </xsl:for-each>       
  </xsl:for-each>
</out>
</xsl:template>
</xsl:stylesheet>