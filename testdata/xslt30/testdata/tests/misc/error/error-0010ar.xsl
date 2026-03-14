<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if an
                     XSLT-defined element is used in a context where it is not permitted, if a
                        required attribute is omitted, or if the content of the
                     element does not correspond to the content that is allowed for the element.-->
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
               xmlns:xs="http://www.w3.org/2001/XMLSchema"
               xmlns:functx="http://functx.com/"
               version="2.0">

   <!-- Test case error190 -->
   <!-- Test that invalid top-level element xsl:garbage is detected -->
   <!-- Author: Michael Kay -->
   
   <?err XTST0010?>

    <xsl:template name="dataSet">
        <xsl:param name="pointValue" as="xs:integer"/>
        <xsl:param name="yScaleFactorRight1" as="xs:integer"/>
        
        <xsl:variable name="coordY" select="$pointValue div $yScaleFactorRight1"/>
        
        <circle cx="10" cy="{-$coordY}" r="4"/>
        
    </xsl:template>
    
    <xsl:template match="/">
        <xsl:copy-of select="functx:wrap-values-in-elements(1 to 10, QName(&#34;http://my.com/&#34;, &#34;ele&#34;))"/>
    </xsl:template>
    
   <xsl:function name="functx:wrap-values-in-elements" as="element()*">
      <xsl:param name="values" as="xs:anyAtomicType*"/> 
      <xsl:param name="elementName" as="xs:QName"/> 
 
      <xsl:for-each select="$values">
         <xsl:element name="{$elementName}"
                      namespace="{namespace-uri-from-QName($elementName)}">
            <xsl:sequence select="."/>
         </xsl:element>
      </xsl:for-each>
 
   </xsl:function>

   <xsl:garbage/>
    
</xsl:transform>
