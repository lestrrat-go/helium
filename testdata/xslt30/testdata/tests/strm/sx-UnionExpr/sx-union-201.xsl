<xsl:stylesheet version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="map xs">
    
    <xsl:variable name="RUN" select="true()" static="yes"/>
    <xsl:strip-space elements="*"/>
 
    <xsl:variable name="extraItem" as="element(ITEM)">
      <ITEM CAT="H">
        <TITLE>Ulysses</TITLE>
        <AUTHOR>James Joyce</AUTHOR>
        <PUBLISHER>HarperCollins</PUBLISHER>
        <PUB-DATE>1935-06-02</PUB-DATE>
        <LANGUAGE>English</LANGUAGE>
        <PRICE>18.90</PRICE>
        <QUANTITY>235</QUANTITY>
        <ISBN>0186701805</ISBN>
        <PAGES>830</PAGES>
        <DIMENSIONS UNIT="in">7.2 5.6 1.2</DIMENSIONS>
        <WEIGHT UNIT="oz">11.0</WEIGHT>
      </ITEM>
    </xsl:variable>
    
 
  <!-- Mixed crawling and grounded sequence. Not streamable: LHS is not a scanning expression 
   (though the rules for scanning expressions could easily be extended to cover this case) -->
  
  <xsl:template name="xsl:initial-template" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:copy-of select="($extraItem | //ITEM)/PRICE"/>
      </out>
    </xsl:source-document>
  </xsl:template> 
            
    
</xsl:stylesheet>