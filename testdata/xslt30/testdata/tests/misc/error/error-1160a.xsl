<?xml version="1.0" encoding="UTF-8"?>
<!--When a URI reference supplied to the
                  document function contains a
                  fragment identifier, it is a recoverable
                     dynamic error if the media type is not one that is recognized
                  by the processor, or if the fragment identifier does not conform to the
                  rules for fragment identifiers for that media type, or if the fragment
                  identifier selects something other than a sequence of nodes (for example, if
                  it selects a range of characters within a text node). The
                     optional recovery
                        action is to ignore the fragment identifier and return the
                     document node.
               -->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTRE1160?>


  <xsl:template name="main">
      <out>
         <xsl:sequence select="document('http://www.w3.org/2005/11/schema-for-xslt20.xsd#123456789')"/>
      </out>
  </xsl:template>
  



</xsl:stylesheet>
