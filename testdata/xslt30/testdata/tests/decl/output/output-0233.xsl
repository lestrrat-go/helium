<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
<!-- Purpose: With HTML output, DOCTYPE must come "immediately before the first element".-->

   <xsl:output method="html" html-version="5" indent="no"/>

   <xsl:template match="/">
      <xsl:comment>This should precede the DOCTYPE declaration</xsl:comment>
      <html>
         <head>
            <title>Title</title>
         </head>
         <body>
            <p>Content</p>
         </body>
      </html>
   </xsl:template>
</xsl:transform>
