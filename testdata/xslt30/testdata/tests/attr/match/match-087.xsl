<?xml version="1.0"?>
<xsl:stylesheet 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    version="3.0">

<?spec xslt#patterns?>
    <!-- Purpose: filtered "/" in pattern -->
  <!-- See spec bug 18861. This is currently allowed in the syntax of patterns, but it is not legal XPath -->
  

<xsl:template match="/[doc]">
 <ok/>
</xsl:template>



</xsl:stylesheet>
