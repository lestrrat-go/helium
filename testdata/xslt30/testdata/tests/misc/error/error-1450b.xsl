<?xml version="1.0" encoding="UTF-8"?>
<!--When a processor performs fallback for
                        an extension instruction
                        that is not recognized, if the instruction element has one or more
                           xsl:fallback children, then the content of each of the
                           xsl:fallback children must be
                        evaluated; it is a non-recoverable
                           dynamic error if it has no xsl:fallback
                        children.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTDE1450?>

  <xsl:template name="main">
    
      <out>        
          <my:inst xsl:extension-element-prefixes="my"/>
      </out>
  </xsl:template>
  



</xsl:stylesheet>
