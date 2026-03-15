<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if an
                     XSLT-defined element is used in a context where it is not permitted, if a
                        required attribute is omitted, or if the content of the
                     element does not correspond to the content that is allowed for the element.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="22.0">

<!-- Note, was error XTDE1450 in an earlier draft of the spec -->

<?error XTSE0010?>

  <xsl:template name="main">
    
      <out>        
          <xsl:banana/>
      </out>
  </xsl:template>
  



</xsl:stylesheet>
