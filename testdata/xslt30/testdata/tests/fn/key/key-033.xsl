<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet version="2.0"
                xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
                
<!-- test key() in an xsl:key, with a variable reference, 
     not the last step in the pattern -->

<xsl:key name="k1" match="*" use="Name/@First"/>
<xsl:key name="k2" match="key('k1', $name)//*" use="@ID"/>
<xsl:param name="name">Ted</xsl:param>
    
<xsl:template match="/">
<out>
  <match><xsl:value-of select="key('k2', 'id8')//Name"/></match>
  <no-match><xsl:value-of select="key('k2', 'id9')//Name"/></no-match>
</out>
</xsl:template>

</xsl:stylesheet>
