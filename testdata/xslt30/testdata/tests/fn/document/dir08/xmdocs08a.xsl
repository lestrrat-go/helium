<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
  
<!-- PURPOSE: Test loading a document relative to the stylesheet module. -->

<xsl:template match="/">
<out>
<xsl:copy-of select="document('./xmdocs08a.xml')"/>
</out>
</xsl:template>
       
</xsl:stylesheet>
