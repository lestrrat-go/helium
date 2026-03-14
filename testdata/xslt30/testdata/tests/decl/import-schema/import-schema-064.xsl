<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
        xmlns:xs="http://www.w3.org/2001/XMLSchema"
        xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
        exclude-result-prefixes="xsi">

<!-- PURPOSE: Show that xsi:type does not cause validation merely by its presence -->
<?spec xslt#validation?>




  <xsl:template match="/">
    <y xsi:type="xs:integer">zzz</y>     
  </xsl:template>
  
</xsl:stylesheet>