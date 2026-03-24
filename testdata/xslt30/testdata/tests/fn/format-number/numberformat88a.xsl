<?xml version="1.0"?>
<?spec xslt#format-number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

   <!-- Purpose: Test overriding of a single attribute of a decimal format,
                 while inheriting other attributes. 
        See Saxon bug http://dev.saxonica.com/community/issues/1571 -->

<xsl:decimal-format name="q" grouping-separator="'" decimal-separator=","/>

</xsl:stylesheet>
