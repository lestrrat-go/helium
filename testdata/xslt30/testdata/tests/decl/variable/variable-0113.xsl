<?xml version="1.0"?>
<xsl:stylesheet version="2.0"
                xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema">
                
<!-- Variable whose value has the wrong item type. Error even in BC mode -->
<?same-as-1.0 no?> 
  <?spec xslt#local-variables?>
<?error ?>           
  
  <xsl:template match="/">
     <xsl:variable name="v" as="xs:integer" select="true()"/>
     <out value="{$v}"/>
     <xsl:message>*** Error not detected! ***</xsl:message>
  </xsl:template>

  
</xsl:stylesheet>
