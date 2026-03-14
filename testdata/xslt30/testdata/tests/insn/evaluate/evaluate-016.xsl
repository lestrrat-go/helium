<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
 >

<!-- Test xsl:evaluate. Formerly test saxon014. -->



<xsl:template match="/">
  <out>
    <xsl:evaluate xpath="//selected/@city" as="element()" context-item="."/>
  </out>
</xsl:template>
 
</xsl:stylesheet>
