<?xml version="1.0" encoding="UTF-8"?>
<!--If an optional parameter has no select attribute and has an empty
                        sequence constructor, and
                     if there is an as attribute, then the default value of the
                     parameter is an empty sequence. If the empty sequence is not a valid instance
                     of the required type defined in the as attribute, then the
                     parameter is treated as a required parameter, which means that it is a non-recoverable dynamic error if the
                     caller supplies no value for the parameter.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTDE0610?>



  <xsl:template name="main">
      <out>
        <xsl:apply-templates>
          
        </xsl:apply-templates>
      </out>
  </xsl:template>
  
  <xsl:template match="doc">
      <xsl:param name="p" as="xs:integer"/>
      <xsl:value-of select="$p"/>
  </xsl:template>   
  


</xsl:stylesheet>
