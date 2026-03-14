<?xml version="1.0" encoding="UTF-8"?>
<!--Where the result of evaluating an XPath expression (or an attribute value
                        template) is required to be a lexical
                           QName, or if it is permitted to be a
                        lexical QName and the actual value
                        takes the form of a lexical QName,
                        then unless otherwise specified it is a non-recoverable dynamic error if the
                        value has a prefix and the
                        defining element has no
                        namespace node whose name matches that prefix. This error
                        may be signaled as a static error if the value of the expression can be determined
                        statically.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTDE1460?>

  <xsl:param name="p" select="'your:qname'"/>

  <xsl:template name="main">
    
      <out>        
          <xsl:result-document href="temp.out" format="{$p}">
               <apple/>
          </xsl:result-document>
      </out>
  </xsl:template>
  
  <xsl:output name="xyz" method="xml"/>


</xsl:stylesheet>
