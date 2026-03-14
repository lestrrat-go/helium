<?xml version="1.0" encoding="UTF-8"?>
<!--A type error occurs
                                 when a document node is validated unless
                              the children of the document node comprise exactly one element node,
                              no text nodes, and zero or more comment and processing instruction
                              nodes, in any order.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTTE1550?>

  <xsl:import-schema namespace="http://err1550a.com/">
      <xs:schema targetNamespace="http://err1550a.com/">
         <xs:element name="e" type="xs:date"/>
      </xs:schema>
  </xsl:import-schema>


  <xsl:template name="main">
      <xsl:result-document validation="strict">
         <xsl:comment>no comment</xsl:comment>
         <e xmlns="http://err1550a.com/">2006-04-01</e> 
      text    
    </xsl:result-document>
  </xsl:template>
  




</xsl:stylesheet>
