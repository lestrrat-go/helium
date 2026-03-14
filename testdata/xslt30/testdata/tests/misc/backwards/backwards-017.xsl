<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="1.0">

  <!-- PURPOSE: effect of BC on default output method -->
  <!-- if XHTML is chosen, meta elements are added and URIs are escaped -->

  <xsl:template name="main">
    <html xmlns="http://www.w3.org/1999/xhtml">
      <head>
        <title>This should not be XHTML</title>
      </head>
      <body>
        <p>This link should not be escaped: <a href="file:///My Documents/page.html">click here</a></p>
      </body>
    </html>  
  </xsl:template>


</xsl:stylesheet>
