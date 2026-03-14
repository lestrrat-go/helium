<?xml version="1.0"?>
<xsl:stylesheet version="2.0"
                xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema">
                
<!-- Variable whose value has the wrong item type --> 
<?spec xslt#local-variables?>
<?error XTTE0570?>           
  
  <xsl:template match="/">
     <xsl:variable name="v" as="xs:integer"><a/></xsl:variable>
     <out att="{$v}"/>
     <xsl:message>*** Error not detected! ***</xsl:message>
  </xsl:template>

  
</xsl:stylesheet>
