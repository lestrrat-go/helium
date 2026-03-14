<?xml version="1.0"?>
<?spec xslt#document?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Purpose: Testing how a relative URI specified in document() is resolved by default. -->
  <!-- Elaboration: "If the second argument [to document()] is omitted,
     then it [the base URI for resolving the relative URI] defaults to
     the node in the stylesheet that contains the expression 
     that includes the call to the document function." The inner call to document, using
     a filename argument rather than a node-set, opens the designated file and gets a node-set
     whose base is that file (or the directory containing it). Further pathing within the
     node-set results in a filename element that contains "level3/xreluri09a.xml" as text.
     The outer call to document() is getting a node-set with a base URI that is not the same as
     this stylesheet nor the reluri11.xml supplied as an argument. Thus, the relative path
     beginning with level3/... (from the inner document call) can only be resolved if
     the outer document() call obtained the base URI of the node-set it was passed as
     its (one) argument. The node-set derived from the file named by the filename has the
     necessary 'body' element. -->

<xsl:template match="doc">
  <out>
    <xsl:copy-of select="document(document('level1/level2/xreluri11b.xml')/*/filename)/*/body"/>
  </out>
</xsl:template>

</xsl:stylesheet>
