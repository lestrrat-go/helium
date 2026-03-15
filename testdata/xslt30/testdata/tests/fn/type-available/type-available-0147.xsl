<xsl:transform
 xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
 xmlns:xs="http://www.w3.org/2001/XMLSchema"
 version="2.0"
>

<!-- Test type-available() (with no schema, but schema-aware processor)  -->


<xsl:template match="/">
<out>
  <integer><xsl:value-of select="type-available('xs:integer')"/></integer>
  <string><xsl:value-of select="type-available('xs:string')"/></string>
  <date><xsl:value-of select="type-available('xs:date')"/></date>
  <time><xsl:value-of select="type-available('xs:time')"/></time>  
  <dayTimeDuration><xsl:value-of select="type-available('xs:dayTimeDuration')"/></dayTimeDuration> 
  <yearMonthDuration><xsl:value-of select="type-available('xs:yearMonthDuration')"/></yearMonthDuration>     
  <untypedAtomic><xsl:value-of select="type-available('xs:untypedAtomic')"/></untypedAtomic>
  <anyType><xsl:value-of select="type-available('xs:anyType')"/></anyType>
  <untyped><xsl:value-of select="type-available('xs:untyped')"/></untyped> 
  <int><xsl:value-of select="type-available('xs:int')"/></int>  
  <short><xsl:value-of select="type-available('xs:short')"/></short>
  <NCName><xsl:value-of select="type-available('xs:NCName')"/></NCName>
  <ID><xsl:value-of select="type-available('xs:ID')"/></ID> 
  <NOTATION><xsl:value-of select="type-available('xs:NOTATION')"/></NOTATION>   
  <unknownType><xsl:value-of select="type-available('xs:unknownType')"/></unknownType>
  <ENTITIES><xsl:value-of select="type-available('xs:ENTITIES')"/></ENTITIES>
  <IDREFS><xsl:value-of select="type-available('xs:IDREFS')"/></IDREFS>  
  <unknownType><xsl:value-of select="type-available('unknownType')"/></unknownType>
  <unknownType><xsl:value-of select="type-available('xsl:unknownType')"/></unknownType>
</out>
</xsl:template>

</xsl:transform>	