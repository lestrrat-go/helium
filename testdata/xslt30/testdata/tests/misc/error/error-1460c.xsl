<?xml version="1.0" encoding="UTF-8"?>
<!--It is a non-recoverable dynamic
                        error if the effective
                        value of the format attribute of an
                           xsl:result-document element is not a valid
                        EQName, or if it does not
                     match the expanded QName of an
                        output definition in the
                        stylesheet. If the processor is able
                     to detect the error statically (for example, when the format
                     attribute contains no curly brackets), then the processor
                        may optionally signal this as a static error.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTDE1460?>

  <xsl:template name="main">
    
      <out>        
          <xsl:result-document format="not:a:qname">
               <apple/>
          </xsl:result-document>
      </out>
  </xsl:template>
  
  <xsl:output name="xyz" method="xml"/>


</xsl:stylesheet>
