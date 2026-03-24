<xsl:transform
 xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
 xmlns:xs="http://www.w3.org/2001/XMLSchema"
 version="2.0"
>

<!-- Test type-available() (with no schema)  -->
<?spec xslt#testing-type-availability?>


<xsl:template match="/">
<out>
  <anyType><xsl:value-of select="type-available('xs:anyType')"/></anyType>
  <integer><xsl:value-of select="type-available('xs:integer')"/></integer>
  <string><xsl:value-of select="type-available('xs:string')"/></string>
  <date><xsl:value-of select="type-available('xs:date')"/></date>
  <time><xsl:value-of select="type-available('xs:time')"/></time>  
  <untypedAtomic><xsl:value-of select="type-available('xs:untypedAtomic')"/></untypedAtomic>
  <anyAtomicType><xsl:value-of select="type-available('xs:anyAtomicType')"/></anyAtomicType>
  <anySimpleType><xsl:value-of select="type-available('xs:anySimpleType')"/></anySimpleType>    
  <untyped><xsl:value-of select="type-available('xs:untyped')"/></untyped> 
  <dayTimeDuration><xsl:value-of select="type-available('xs:dayTimeDuration')"/></dayTimeDuration> 
  <yearMonthDuration><xsl:value-of select="type-available('xs:yearMonthDuration')"/></yearMonthDuration>     
  <unknownType><xsl:value-of select="type-available('xs:unknownType')"/></unknownType>
  <unknownType><xsl:value-of select="type-available('unknownType')"/></unknownType>
  <unknownType><xsl:value-of select="type-available('xsl:unknownType')"/></unknownType>
</out>
</xsl:template>

</xsl:transform>	