<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="1.0">

  <!-- PURPOSE: effect of BC on default output method -->
  <!-- if XHTML is chosen, meta elements are added and URIs are escaped -->
  
  <!--  "If the expanded QName of this first element child has local part html (in lower case), and namespace URI http://www.w3.org/1999/xhtml, 
         then the default output method is normally xhtml. However, if the effective version of the outermost element of the principal stylesheet 
         module in the top-level package has the value 1.0, and if the result tree is generated implicitly (rather than by an explicit xsl:result-document 
         instruction), then the default output method in this situation is xml. "-->

  <xsl:template name="main">
    <html xmlns="http://www.w3.org/1999/xhtml">
      <head>
        <title>In 1.0 mode this should be XML (not XHTML)</title>
      </head>
      <body>
        <p>This link should therefore not be escaped: <a href="file:///My D&#xf8;cuments/page.html">click here</a></p>
      </body>
    </html>     
  </xsl:template>


</xsl:stylesheet>
