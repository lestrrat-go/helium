<xsl:stylesheet version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:err="http://www.w3.org/2005/xqt-errors"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="map xs err">
    
    <xsl:variable name="RUN" select="true()" static="yes"/>
    <xsl:strip-space elements="*"/>

    
    <!-- Test of xsl:source-document with sum() -->
    
    <xsl:template name="s-003" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="sum(./BOOKLIST/BOOKS/ITEM/PRICE)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with sum(), filtered with a motionless predicate -->
    
    <xsl:template name="s-004" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="sum(./BOOKLIST/BOOKS/ITEM[@CAT='P']/PRICE)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with sum(), filtered with a positional predicate -->
    
    <xsl:template name="s-005" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="sum(./BOOKLIST/BOOKS/ITEM[position() lt 4]/PRICE)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with sum(), filtered with a positional predicate -->
    
    <xsl:template name="s-006" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="sum(./BOOKLIST/BOOKS/*[position() lt 4]/PRICE)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with sum(), filtered with a positional predicate -->
    
    <xsl:template name="s-007" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:sequence select="sum(./BOOKLIST/BOOKS/*:ITEM[position() lt 4]/PRICE)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with sum(), filtered using data() to make it streamable -->
    
    <xsl:template name="s-008" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="round(sum(./BOOKLIST/BOOKS/ITEM/PAGES/data()[. &lt; 1000][. &gt; 0]))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with sum(), filtered using text() to make it streamable -->
    
    <xsl:template name="s-009" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="round(sum(./BOOKLIST/BOOKS/ITEM/PAGES/text()[. &lt; 1000][. &gt; 0]))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with sum(), filtered using number() to make it streamable -->
    
    <xsl:template name="s-010" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="round(sum(./BOOKLIST/BOOKS/ITEM/PAGES/number()[. &lt; 1000][. &gt; 0]))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with sum(), selecting nothing and returning the $zero result -->
    
    <xsl:template name="s-011" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="sum(ITEM/PAGES, 42)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of sum() selecting both streamed nodes and literals -->
    
    <xsl:template name="s-012" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="sum((./BOOKLIST/BOOKS/ITEM/PAGES/number(), 31, 32))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of sum() selecting both streamed nodes and literals while also filtering -->
    
    <xsl:template name="s-013" use-when="true() or $RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="sum((tail(./BOOKLIST/BOOKS/ITEM/PAGES)/number(), 31, 32))"/>
        </out>
      </xsl:source-document>
    </xsl:template>   
    
    <!-- Test of xsl:source-document with sum of a computed value -->
    
    <xsl:template name="s-015" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="format-number(sum(for $d in data(outermost(//DIMENSIONS))
                                                  return let $x := tokenize($d, '\s')!number() 
                                                  return $x[1]*$x[2]*$x[3]), '99.999')"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with sum of a computed value -->
    
    <xsl:template name="s-016" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <xsl:value-of select="xs:integer(round(sum(account/transaction/(@value*2))))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with sum(), filtered using copy-of() to make it streamable -->
    
    <xsl:template name="s-017" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="round(sum(./BOOKLIST/BOOKS/ITEM/PAGES/copy-of()[. &lt; 1000][. &gt; 0]))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with sum() applied to an attribute -->
    
    <xsl:template name="s-018" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <xsl:value-of select="round(sum(account/transaction/@value))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with sum() over attribute values, filtered -->
    
    <xsl:template name="s-019" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <xsl:value-of select="round(sum(account/transaction/@value[xs:decimal(.) gt 0]))"/>
        </out>
      </xsl:source-document>
    </xsl:template> 
    
    <!-- Test of xsl:source-document with sum() over attribute values, computed -->
    
    <xsl:template name="s-020" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <xsl:value-of select="round(sum(account/transaction/abs(@value)))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- sum() of a conditional value -->
    
    <xsl:template name="s-021" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <xsl:value-of select="round(sum(account/transaction/
                (if (xs:date(@date) lt xs:date('2020-01-01')) then +@value else (@value+1))))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- sum() with a loop used to compute each item -->
    
    <xsl:template name="s-022" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="sum(BOOKLIST/BOOKS/ITEM/DIMENSIONS!xs:NMTOKENS(.)!xs:decimal(.))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document calling sum() using //x/y -->
    <!-- Saxon makes this streamable by rewriting //X/Y as .//Y[parent::X] -->
    
    <xsl:template name="s-023" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="sum(/BOOKLIST/BOOKS/ITEM/DIMENSIONS!xs:NMTOKENS(.)!xs:decimal(.))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming sum(): grounded operand, selects nothing -->
    
    <xsl:template name="s-040" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="sum(data(/BOOKLIST/BOOKS/NOTHING), -1)" separator="|"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming sum(): striding operand, selects nothing -->
    
    <xsl:template name="s-041" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="sum(/BOOKLIST/BOOKS/MAGAZINE, -1)" separator="|"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming sum(): crawling operand, selects nothing -->
    
    <xsl:template name="s-042" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="sum(/BOOKLIST/BOOKS/MAGAZINE, -1)" separator="|"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming sum(): climbing operand, selects nothing -->
    
    <xsl:template name="s-043" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="sum(//PRICE/../@nothing, -1)" separator="|"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming sum(): second operand is consuming (but unused) -->
    
    <xsl:template name="s-044" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="sum(1 to 10, number(head(//PRICE)))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming sum(): second operand is consuming (and unused) -->
    
    <xsl:template name="s-045" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="sum((1 to 10)[current-date() lt xs:date('1999-11-16')], number(head(//PRICE)))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming sum(): dayTimeDuration values -->
    
    <xsl:template name="s-050" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="sum(/BOOKLIST/BOOKS/ITEM/PUB-DATE ! (xs:date('2050-01-01') - xs:date(.)))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming sum(): yearMonthDuration values -->
    
    <xsl:template name="s-051" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="sum(
                                    for $pubdate in /BOOKLIST/BOOKS/ITEM/PUB-DATE/xs:date(.) return
                                    let $monthsAgo := (2050*12) - (12*year-from-date($pubdate) + month-from-date($pubdate)) return
                                    xs:yearMonthDuration('P1M') * $monthsAgo)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming sum(): inconsistent type values -->
    
    <xsl:template name="s-052" use-when="$RUN">
      <xsl:param name="p" select="23"/>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="sum(($p, /BOOKLIST/BOOKS/ITEM/PUB-DATE ! (xs:date('2050-01-01') - xs:date(.))))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming sum(): incorrect type values -->
    
    <xsl:template name="s-053" use-when="$RUN">
      <xsl:param name="p" select="'Wrong'"/>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="sum(($p, /BOOKLIST/BOOKS/ITEM/PUB-DATUM ! (xs:date('2050-01-01') - xs:date(.))))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming sum(): inconsistent type values, error is caught -->
    
    <xsl:template name="s-054" use-when="$RUN">
      <xsl:param name="p" select="23"/>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:try>
            <xsl:value-of select="sum(($p, /BOOKLIST/BOOKS/ITEM/PUB-DATE ! (xs:date('2050-01-01') - xs:date(.))))"/>
            <xsl:catch errors="err:FORG0006" select="'caught'"/>
          </xsl:try>  
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming sum(): incorrect type values, error is caught -->
    
    <xsl:template name="s-055" use-when="$RUN">
      <xsl:param name="p" select="'Wrong'"/>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:try>
            <xsl:value-of select="sum(($p, /BOOKLIST/BOOKS/ITEM/PUB-DATUM ! (xs:date('2050-01-01') - xs:date(.))))"/>
            <xsl:catch errors="err:FORG0006" select="'caught'"/>
          </xsl:try>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming sum(): crawling operand including nested nodes -->
    
    <xsl:template name="s-056" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/nested-numbers.xml">
        <out>
          <xsl:value-of select="sum(descendant::n)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
  
  <!-- Streaming sum(): crawling operand, second argument is ignored when first is non-empty (bug 29420) -->
  
  <xsl:template name="s-057" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of select="sum(//PRICE, 123)" separator="|"/>
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <!-- Streaming sum(): first operand grounded and motionless, second argument consuming -->
  
  <xsl:template name="s-058" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of select="sum((1 to 5)[. gt year-from-date(current-date())], sum(//PRICE))" separator="|"/>
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <!-- Streaming sum(): first operand grounded and motionless, second argument consuming, but not actually evaluated -->
  
  <xsl:template name="s-059" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of select="sum((1 to 5)[. lt year-from-date(current-date())], sum(//PRICE))" separator="|"/>
      </out>
    </xsl:source-document>
  </xsl:template>
                         
                                                
    
</xsl:stylesheet>