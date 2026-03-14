<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if an
                        xsl:attribute-set element directly or indirectly references
                     itself via the names contained in the use-attribute-sets
                     attribute.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTSE0720?>



  <xsl:template match="/">
      <out>
        <xsl:apply-templates/>
        <xsl:element name="e" use-attribute-sets="a"/>
      </out>
  </xsl:template>
  
  <xsl:attribute-set name="a" use-attribute-sets="a"/>
  



</xsl:stylesheet>
