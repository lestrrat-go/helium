<?xml version="1.0" encoding="UTF-8"?>
<!--It is a non-recoverable dynamic
                                 error if the result sequence used to construct the
                              content of a document node contains a namespace node or attribute
                              node.-->
<!-- this succeeds in 1.0, with top-down construction, but is should fail in 2.0 with bottom up construction.
      Note that the xsl:copy is copying a document node which can't have an attribute. -->
<?spec xslt#errors?>
<?error XTDE0420?>
<a xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xsl:version="2.0">
   <xsl:copy>
      <xsl:attribute name="x">5</xsl:attribute>
   </xsl:copy>
</a>
