<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0" xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">
 <xsl:output method="xml" encoding="UTF-8" indent="no" omit-xml-declaration="yes" />
 <xsl:template match="/">
<x>
  <!-- Testing fn:node-name#0 -->
   <?spec fo#func-node-name?>
  <xsl:for-each select="//*">
   <out>
   <xsl:value-of select="node-name()"/>
   </out>
  </xsl:for-each>
     
</x>
 </xsl:template>
</xsl:stylesheet>
